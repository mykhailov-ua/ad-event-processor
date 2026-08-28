package controlplane

import "ad-event-processor/internal/brand"

type BrandDTO = brand.DTO
type BrandCreativeDTO = brand.CreativeDTO
type CreateBrandRequest = brand.CreateRequest
type UpsertBrandCreativeRequest = brand.UpsertCreativeRequest
type UpdateBrandCreativeRequest = brand.UpdateCreativeRequest

type BrandHTTPHandlers = brand.HTTPHandlers
