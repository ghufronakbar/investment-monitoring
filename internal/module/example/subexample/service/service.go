// internal/module/example/service/service.go
package subexample_service

import (
	"context"
	"database/sql"
	"investment_monitoring/internal/repository/entity"
	"investment_monitoring/pkg/errs"
	"investment_monitoring/pkg/pagination"

	"github.com/google/uuid"
)

type SubExampleService struct {
	store entity.Store
}

func NewService(store entity.Store) *SubExampleService {
	return &SubExampleService{
		store: store,
	}
}

func (s *SubExampleService) GetList(ctx context.Context, filterData SubExampleGetListParams) ([]SubExampleGetReturn, *pagination.Pagination, error) {
	data, err := s.store.GetListExample(ctx, entity.GetListExampleParams{
		Search:  filterData.Search,
		SortBy:  filterData.SortBy,
		SortDir: filterData.SortDir,
		Offset:  int32(filterData.Offset),
		Limit:   int32(filterData.Limit),
	})
	if err != nil {
		return nil, nil, err
	}
	total, err := s.store.CountExample(ctx, filterData.Search)
	if err != nil {
		return nil, nil, err
	}
	paginationParams := pagination.PaginationParams{
		Total: int(total),
		Page:  filterData.Page,
		Limit: filterData.Limit,
	}

	pagination := pagination.NewPagination(&paginationParams)

	returnData := make([]SubExampleGetReturn, 0, len(data))

	for _, item := range data {
		returnData = append(returnData, SubExampleGetReturn{
			ID:        item.ID,
			Name:      item.Name,
			Price:     item.Price,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}

	return returnData, &pagination, nil
}

func (s *SubExampleService) GetDetail(ctx context.Context, id uuid.UUID) (SubExampleDetailReturn, error) {
	data, err := s.store.GetExampleByID(ctx, id)
	if err == sql.ErrNoRows {
		return SubExampleDetailReturn{}, errs.NewNotFound("")
	}
	if err != nil {
		return SubExampleDetailReturn{}, err
	}
	return SubExampleDetailReturn{
		ID:        data.ID,
		Name:      data.Name,
		Price:     data.Price,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}, nil
}

func (s *SubExampleService) Create(ctx context.Context, input SubExampleCreateParams) (SubExampleGetReturn, error) {
	data, err := s.store.CreateExample(ctx, entity.CreateExampleParams{
		Name:  input.Name,
		Price: input.Price,
	})
	if err != nil {
		return SubExampleGetReturn{}, err
	}
	return SubExampleGetReturn{
		ID:        data.ID,
		Name:      data.Name,
		Price:     data.Price,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}, nil
}

func (s *SubExampleService) Edit(ctx context.Context, input SubExampleEditParams) (SubExampleGetReturn, error) {
	data, err := s.store.UpdateExampleByID(ctx, entity.UpdateExampleByIDParams{
		ID:    input.ID,
		Name:  input.Name,
		Price: input.Price,
	})
	if err == sql.ErrNoRows {
		return SubExampleGetReturn{}, errs.NewNotFound("")
	}
	if err != nil {
		return SubExampleGetReturn{}, err
	}
	return SubExampleGetReturn{
		ID:    data.ID,
		Name:  data.Name,
		Price: data.Price,
	}, nil
}

func (s *SubExampleService) Delete(ctx context.Context, input SubExampleDeleteParams) (SubExampleDeleteReturn, error) {
	data, err := s.store.SoftDeleteExampleByID(ctx, input.ID)
	if err == sql.ErrNoRows {
		return SubExampleDeleteReturn{}, errs.NewNotFound("")
	}
	if err != nil {
		return SubExampleDeleteReturn{}, err
	}
	return SubExampleDeleteReturn{
		ID:        data.ID,
		Name:      data.Name,
		Price:     data.Price,
		CreatedAt: data.CreatedAt,
		UpdatedAt: data.UpdatedAt,
	}, nil
}
