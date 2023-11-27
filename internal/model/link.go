package model

type LinkType string

const (
	ThingModelMediaType LinkType = "application/tm+json"
)

type RelType string

const (
	RelContent RelType = "content"
)

type Link struct {
	Rel  RelType  `json:"rel"`
	HRef string   `json:"href"`
	Type LinkType `json:"type,omitempty"`
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
