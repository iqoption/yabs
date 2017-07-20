package base

import (
	"fmt"
	"context"
	"yabs/common/format/minidump"
	"encoding/json"
	"reflect"
	"gopkg.in/olivere/elastic.v5"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Repository struct {
	db    *elastic.Client
	cache Cashe
}

type Symbol struct {
	DirPath   string `json:"path"`
	Version   string `json:"build"`
	DebugId   string `json:"debugId"`
	Platform  string `json:"platform"`
	DataAdded string `json:"date_added"`
}

func (r *Repository) putInCacheVersion2Id(v string, i uint64) {
	err := r.cache.Set(v, fmt.Sprintf("%d", i))
	if err != nil {
		log.WithError(err).Warning("Can't append build info in cache")
	}
}

func (r *Repository) putInCacheDebugId2Version(debId, ver string) {
	err := r.cache.Set(debId, ver)
	if err != nil {
		log.WithError(err).Warning("Can't put in cache debug_id from symbol")
	}
}

func (r *Repository) AddSymbol(s *Symbol) error {
	_, err := r.db.
		Index().
		Index("breakpad").
		Type("symbol").
		Id(s.DebugId).
		BodyJson(s).
		Refresh("true").
		Do(context.Background())

	if err == nil {
		r.putInCacheDebugId2Version(s.DebugId, s.Version)
	}

	return err
}

func (r *Repository) IsExist(s *Symbol) (bool, error) {
	ver := r.getFromCacheVersion(s.DebugId)
	if ver != "" {
		return true, nil
	}

	ss, err := r.GetSymbol(s.DebugId)
	if (ss != nil) && (err == nil) {
		return true, nil
	}

	return false, nil
}

func (r *Repository) getFromCacheVersion(debugId string) string {
	v, err := r.cache.Get(debugId)
	if err != nil {
		return ""
	}

	return v
}

func (r *Repository) GetSymbol(debugId string) (*Symbol, error) {
	ver := r.getFromCacheVersion(debugId)
	if ver != "" {
		return &Symbol{
			Version: ver,
			DebugId: debugId,
		}, nil
	}

	get, err := r.db.Get().
		Index("breakpad").
		Type("symbol").
		Id(debugId).
		Do(context.Background())

	if err == nil {
		var s Symbol
		err = json.Unmarshal(*get.Source, &s)
		if err != nil {
			log.WithError(err).Error("Can't deserialize symbol")
			return nil, err
		}
		s.DebugId = debugId
		r.putInCacheDebugId2Version(debugId, s.Version)
		return &s, nil
	}

	return nil, err
}

func (r *Repository) GetSymbolForPlatform(platform, version string) (*Symbol, error) {
	filter := elastic.NewBoolQuery().Must(elastic.NewTermQuery("platform", platform),
		elastic.NewTermQuery("build", version))
	query := elastic.NewConstantScoreQuery(filter)

	searchRes, err := r.db.Search().
		Index("breakpad").
		Query(query).
		Type("symbol").
		Do(context.Background())
	if err != nil {
		log.WithFields(log.Fields{
			"platform": platform,
			"version":  version,
			"error":    err,
		}).Error("Can't search symbols on platform and version")
		return nil, err
	}

	var s Symbol

	for _, item := range searchRes.Each(reflect.TypeOf(s)) {
		symbol := item.(Symbol)
		return &symbol, nil
	}

	return nil, nil
}

func (r *Repository) AddReport(report *minidump.Report) (string, error) {

	uuid := uuid.NewV4().String()
	_, err := r.db.
		Index().
		Index("breakpad").
		Type("crash").
		Id(uuid).
		BodyJson(report).
		Refresh("true").
		Do(context.Background())

	if err != nil {
		log.WithError(err).Error("Can't insert crash report")
	}

	return uuid, err
}

func NewRepository(connectionUrl string, c Cashe) (*Repository, error) {
	b, err := elastic.NewClient(elastic.SetURL(connectionUrl))
	return &Repository{
		db:    b,
		cache: c,
	}, err
}
