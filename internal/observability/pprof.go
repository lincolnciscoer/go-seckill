package observability

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

func RegisterPprofRoutes(engine *gin.Engine) {
	engine.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	engine.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	engine.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	engine.POST("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	engine.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	engine.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
	engine.GET("/debug/pprof/allocs", gin.WrapH(pprof.Handler("allocs")))
	engine.GET("/debug/pprof/block", gin.WrapH(pprof.Handler("block")))
	engine.GET("/debug/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	engine.GET("/debug/pprof/heap", gin.WrapH(pprof.Handler("heap")))
	engine.GET("/debug/pprof/mutex", gin.WrapH(pprof.Handler("mutex")))
	engine.GET("/debug/pprof/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
}
