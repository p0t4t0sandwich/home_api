package wishlist

type WishListItem struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	Price    float64 `json:"price"`
	Currency string  `json:"currency"`
}

type WishList struct {
	Items []WishListItem `json:"items"`
}

func NewWishList() *WishList {
	return &WishList{}
}

func (wl *WishList) AddItem(item WishListItem) {
	wl.Items = append(wl.Items, item)
}

func (wl *WishList) RemoveItem(id int) {
	for i, item := range wl.Items {
		if item.ID == id {
			wl.Items = append(wl.Items[:i], wl.Items[i+1:]...)
			return
		}
	}
}
