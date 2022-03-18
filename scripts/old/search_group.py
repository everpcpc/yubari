from elasticsearch import Elasticsearch
from elasticsearch_dsl import Search

es = Elasticsearch("http://localhost:9200/")

IDX = "supergroup--10011111111111"


query_body = {
    "aggs": {
        "by_id": {
            "filter": {
                "range": {
                    "id": {"gte": 0},
                }
            },
            "aggs": {
                "by_user": {
                    "terms": {"field": "user"},
                }
            },
        },
    }
}


s = Search.from_dict(query_body).using(es).index(IDX)
t = s.execute()
for item in t.aggregations.by_id.by_user.buckets:
    print(item.key, item.doc_count)
