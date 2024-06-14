package model

type RelType string

const (
	RelContent RelType = "content"
)

type Link struct {
	Rel  RelType `json:"rel"`
	HRef string  `json:"href"`
}

type Links []Link

func (links *Links) FindLink(rel RelType) *Link {
	for _, link := range *links {
		if link.Rel == rel {
			return &link
		}
	}
	return nil
}
