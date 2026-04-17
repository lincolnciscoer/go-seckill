package cache

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	SeckillAllowed        int64 = 0
	SeckillSoldOut        int64 = 1
	SeckillRepeatOrder    int64 = 2
	SeckillActivityClosed int64 = 3
	SeckillNotStarted     int64 = 4
	SeckillEnded          int64 = 5
	SeckillNotPreheated   int64 = 6
)

var seckillAttemptScript = goredis.NewScript(`
local stockKey = KEYS[1]
local userKey = KEYS[2]

local status = tonumber(ARGV[1])
local nowTs = tonumber(ARGV[2])
local startTs = tonumber(ARGV[3])
local endTs = tonumber(ARGV[4])
local markerTTL = tonumber(ARGV[5])

if status ~= 1 then
	return {3, -1}
end

if nowTs < startTs then
	return {4, -1}
end

if nowTs > endTs then
	return {5, -1}
end

if redis.call("EXISTS", userKey) == 1 then
	local currentStock = redis.call("GET", stockKey)
	if currentStock then
		return {2, tonumber(currentStock)}
	end
	return {2, -1}
end

local stock = redis.call("GET", stockKey)
if not stock then
	return {6, -1}
end

stock = tonumber(stock)
if stock <= 0 then
	return {1, stock}
end

local remainingStock = redis.call("DECR", stockKey)
redis.call("SET", userKey, "1", "EX", markerTTL)
return {0, tonumber(remainingStock)}
`)

var seckillCompensateScript = goredis.NewScript(`
local stockKey = KEYS[1]
local userKey = KEYS[2]

if redis.call("EXISTS", stockKey) == 1 then
	redis.call("INCR", stockKey)
end

redis.call("DEL", userKey)
return 1
`)

type SeckillAttemptInput struct {
	ActivityID uint64
	UserID     uint64
	Status     int8
	Now        time.Time
	StartTime  time.Time
	EndTime    time.Time
}

type SeckillAttemptResult struct {
	Code           int64
	RemainingStock int64
}

// AttemptSeckill 用 Lua 脚本在 Redis 中原子完成：
// 1. 活动状态校验
// 2. 活动时间窗口校验
// 3. 一人一单判断
// 4. 库存预扣减
func (c *ActivityCache) AttemptSeckill(ctx context.Context, input SeckillAttemptInput) (*SeckillAttemptResult, error) {
	markerTTL := int64(time.Until(input.EndTime).Seconds()) + 3600
	if markerTTL < 1 {
		markerTTL = 1
	}

	result, err := seckillAttemptScript.Run(
		ctx,
		c.client,
		[]string{
			activityStockKey(input.ActivityID),
			activityUserKey(input.ActivityID, input.UserID),
		},
		input.Status,
		input.Now.Unix(),
		input.StartTime.Unix(),
		input.EndTime.Unix(),
		markerTTL,
	).Result()
	if err != nil {
		return nil, err
	}

	values, ok := result.([]any)
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("unexpected lua result: %#v", result)
	}

	code, err := parseLuaInteger(values[0])
	if err != nil {
		return nil, err
	}

	remainingStock, err := parseLuaInteger(values[1])
	if err != nil {
		return nil, err
	}

	return &SeckillAttemptResult{
		Code:           code,
		RemainingStock: remainingStock,
	}, nil
}

func (c *ActivityCache) CompensateSeckill(ctx context.Context, activityID uint64, userID uint64) error {
	return seckillCompensateScript.Run(
		ctx,
		c.client,
		[]string{
			activityStockKey(activityID),
			activityUserKey(activityID, userID),
		},
	).Err()
}

func (c *ActivityCache) InvalidateActivityViews(ctx context.Context, activityID uint64) error {
	return c.client.Del(ctx, activityListKeyPrefix, activityDetailKey(activityID)).Err()
}

func activityUserKey(activityID uint64, userID uint64) string {
	return fmt.Sprintf("seckill:user:%d:%d", activityID, userID)
}

func parseLuaInteger(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unexpected lua integer type: %T", value)
	}
}
