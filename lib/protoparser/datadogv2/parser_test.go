package datadogv2

import (
	"reflect"
	"testing"
)

func TestRequestUnmarshalJSONFailure(t *testing.T) {
	f := func(s string) {
		t.Helper()
		var req Request
		if err := UnmarshalJSON(&req, []byte(s)); err == nil {
			t.Fatalf("expecting non-nil error for Unmarshal(%q)", s)
		}
	}
	f("")
	f("foobar")
	f(`{"series":123`)
	f(`1234`)
	f(`[]`)
}

func TestRequestUnmarshalJSONSuccess(t *testing.T) {
	f := func(s string, reqExpected *Request) {
		t.Helper()
		var req Request
		if err := UnmarshalJSON(&req, []byte(s)); err != nil {
			t.Fatalf("unexpected error in Unmarshal(%q): %s", s, err)
		}
		if !reflect.DeepEqual(&req, reqExpected) {
			t.Fatalf("unexpected row;\ngot\n%+v\nwant\n%+v", &req, reqExpected)
		}
	}
	f("{}", &Request{})
	f(`
{
  "series": [
    {
      "metric": "system.load.1",
      "type": 0,
      "points": [
        {
          "timestamp": 1636629071,
          "value": 0.7
        }
      ],
      "resources": [
        {
          "name": "dummyhost",
          "type": "host"
        }
      ],
      "source_type_name": "kubernetes",
      "tags": ["environment:test"]
    }
  ]
}
`, &Request{
		Series: []Series{{
			Metric: "system.load.1",
			Points: []Point{
				{
					Timestamp: 1636629071,
					Value:     0.7,
				},
			},
			Resources: []Resource{
				{
					Name: "dummyhost",
					Type: "host",
				},
			},
			SourceTypeName: "kubernetes",
			Tags: []string{
				"environment:test",
			},
		}},
	})
	f(`
{
  "series": [
    {
      "metric": "demo.rate",
      "type": 2,
      "interval": 10,	
      "points": [
        {
          "timestamp": 1636629071,
          "value": 0.7
        }
      ],
      "resources": [
        {
          "name": "dummyhost",
          "type": "host"
        }
      ],
      "source_type_name": "kubernetes",
      "tags": ["environment:test"]
    }
  ]
}
`, &Request{
		Series: []Series{{
			Metric:   "demo.rate",
			Type:     2,
			Interval: 10,
			Points: []Point{
				{
					Timestamp: 1636629071,
					Value:     0.7,
				},
			},
			Resources: []Resource{
				{
					Name: "dummyhost",
					Type: "host",
				},
			},
			SourceTypeName: "kubernetes",
			Tags: []string{
				"environment:test",
			},
		}},
	})
}

// A request is reused for the next body. A protobuf body that stops on
// an error after its first series got a metric name, a tag and a point
// must not give them to a later JSON series that has none of them.
func TestRequestReuseAfterProtobufFailure(t *testing.T) {
	var req Request
	if err := UnmarshalJSON(&req, []byte(`{"series":[{"metric":"a","tags":["x:y"],"points":[{"timestamp":1,"value":2}]}]}`)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Series 1 holds metric "r", tag "t" and a point with value 5, and
	// then field 4 (points) with the varint wire type, which the decoder
	// rejects.
	body := []byte{
		0x0a, 0x13,
		0x12, 0x01, 'r',
		0x1a, 0x01, 't',
		0x22, 0x09, 0x09, 0, 0, 0, 0, 0, 0, 0x14, 0x40,
		0x20, 0x01,
	}
	if err := UnmarshalProtobuf(&req, body); err == nil {
		t.Fatalf("expecting non-nil error for the protobuf body")
	}

	if err := UnmarshalJSON(&req, []byte(`{"series":[{"points":[{}]}]}`)); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(req.Series) != 1 {
		t.Fatalf("unexpected number of series: %d", len(req.Series))
	}
	s := req.Series[0]
	if s.Metric != "" {
		t.Fatalf("metric name of an earlier request: %q", s.Metric)
	}
	if len(s.Tags) != 0 {
		t.Fatalf("tags of an earlier request: %q", s.Tags)
	}
	if len(s.Points) != 1 || s.Points[0].Value != 0 {
		t.Fatalf("point of an earlier request: %+v", s.Points)
	}
}
