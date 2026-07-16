package server

import (
	"context"
	"log"

	"github.com/samber/lo"
)

type server struct{}

func NewServer() StrictServerInterface { return &server{} }

func (*server) SearchMetadata(
	ctx context.Context,
	request SearchMetadataRequestObject,
) (SearchMetadataResponseObject, error) {
	books, err := searchMetadataBooks(ctx, request.Params.Query, request.Params.Author)
	if err != nil {
		log.Printf("SearchMetadata error: %v", err)
		return SearchMetadata500JSONResponse{N500JSONResponse{Error: lo.ToPtr(err.Error())}}, nil
	}

	return SearchMetadata200JSONResponse{N200JSONResponse{Matches: &books}}, nil
}
