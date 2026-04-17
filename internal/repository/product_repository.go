package repository

import (
	"context"
	"errors"

	"go-seckill/internal/model"

	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(ctx context.Context, product *model.Product) error
	List(ctx context.Context) ([]model.Product, error)
	GetByID(ctx context.Context, id uint64) (*model.Product, error)
}

type GormProductRepository struct {
	db *gorm.DB
}

func NewGormProductRepository(db *gorm.DB) *GormProductRepository {
	return &GormProductRepository{db: db}
}

func (r *GormProductRepository) Create(ctx context.Context, product *model.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *GormProductRepository) List(ctx context.Context) ([]model.Product, error) {
	var products []model.Product
	err := r.db.WithContext(ctx).Order("id DESC").Find(&products).Error
	return products, err
}

func (r *GormProductRepository) GetByID(ctx context.Context, id uint64) (*model.Product, error) {
	var product model.Product
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &product, nil
}
