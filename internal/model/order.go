package model

type Order struct {
    ID       int64  `json:"id"`
    Distance int    `json:"distance"`
    Status   string `json:"status"`
}

type PlaceOrderRequest struct {
    Origin      []string `json:"origin"`
    Destination []string `json:"destination"`
}

type TakeOrderRequest struct {
    Status string `json:"status"`
}

type ErrorResponse struct {
    Error string `json:"error"`
}

type TakeOrderResponse struct {
    Status string `json:"status"`
}