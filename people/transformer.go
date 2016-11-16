package people

import (
	"encoding/base64"
	"encoding/xml"
	"github.com/pborman/uuid"
)

func transformPerson(tmeTerm term, taxonomyName string) person {
	tmeIdentifier := buildTmeIdentifier(tmeTerm.RawID, taxonomyName)
	personUUID := uuid.NewMD5(uuid.UUID{}, []byte(tmeIdentifier)).String()
	return person{
		UUID:      personUUID,
		PrefLabel: tmeTerm.CanonicalName,
		AlternativeIdentifiers: alternativeIdentifiers{
			TME:   []string{tmeIdentifier},
			Uuids: []string{personUUID},
		},
		Type: "Person",
	}
}

func buildTmeIdentifier(rawID string, tmeTermTaxonomyName string) string {
	id := base64.StdEncoding.EncodeToString([]byte(rawID))
	taxonomyName := base64.StdEncoding.EncodeToString([]byte(tmeTermTaxonomyName))
	return id + "-" + taxonomyName
}

type personTransformer struct {
}

func (*personTransformer) UnMarshallTaxonomy(contents []byte) ([]interface{}, error) {
	t := taxonomy{}
	err := xml.Unmarshal(contents, &t)
	if err != nil {
		return nil, err
	}
	interfaces := make([]interface{}, len(t.Terms))
	for i, d := range t.Terms {
		interfaces[i] = d
	}
	return interfaces, nil
}

func (*personTransformer) UnMarshallTerm(content []byte) (interface{}, error) {
	dummyTerm := term{}
	err := xml.Unmarshal(content, &dummyTerm)
	if err != nil {
		return term{}, err
	}
	return dummyTerm, nil
}
