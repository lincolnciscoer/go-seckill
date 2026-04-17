package service

import (
	"context"
	"strings"

	"go-seckill/internal/model"
	"go-seckill/internal/repository"
)

type ProductService struct {
	products repository.ProductRepository
}

type CreateProductInput struct {
	Name        string
	Description string
	Price       int64
	Status      int8
}

func NewProductService(products repository.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

func (s *ProductService) Create(ctx context.Context, input CreateProductInput) (*model.Product, error) {
	product := &model.Product{
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Price:       input.Price,
		Status:      input.Status,
	}

	if product.Status == 0 {
		product.Status = 1
	}

	if err := s.products.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (s *ProductService) List(ctx context.Context) ([]model.Product, error) {
	return s.products.List(ctx)
}
