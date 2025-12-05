package model

type AddressSnapshot struct {
	AddressID uint32 `json:"address_id"`
	Name      string `json:"name"`
	Desc      string `json:"desc"`
}

type ProductSnapshot struct {
	ProductID  uint32 `json:"product_id"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	Qty        uint32 `json:"qty"`
	Subtotal   int64  `json:"subtotal"`
	CategoryID uint32 `json:"category_id"`
}
