package dlna

import (
	"encoding/xml"
	"fmt"
	"net/url"
)

// servedHeaders are the headers a DLNA device expects on a response its sink entry describes.
func servedHeaders(entry sinkEntry) map[string]string {
	headers := map[string]string{
		"Connection":            "close",
		"Accept-Ranges":         "none",
		"transferMode.dlna.org": "Streaming",
	}
	if entry.features != "*" {
		headers["contentFeatures.dlna.org"] = entry.features
	}
	return headers
}

type didlLite struct {
	XMLName xml.Name `xml:"DIDL-Lite"`
	XMLNS   string   `xml:"xmlns,attr"`
	DC      string   `xml:"xmlns:dc,attr"`
	UPnP    string   `xml:"xmlns:upnp,attr"`
	Item    didlItem `xml:"item"`
}

type didlItem struct {
	ID         string  `xml:"id,attr"`
	ParentID   string  `xml:"parentID,attr"`
	Restricted string  `xml:"restricted,attr"`
	Title      string  `xml:"dc:title"`
	Class      string  `xml:"upnp:class"`
	Res        didlRes `xml:"res"`
}

type didlRes struct {
	ProtocolInfo string `xml:"protocolInfo,attr"`
	Value        string `xml:",chardata"`
}

// buildDIDLMetadata returns the DIDL-Lite XML announcing streamURL as the device's own sink entry describes it.
func buildDIDLMetadata(streamURL *url.URL, entry sinkEntry) (string, error) {
	item := didlLite{
		XMLNS: "urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/",
		DC:    "http://purl.org/dc/elements/1.1/",
		UPnP:  "urn:schemas-upnp-org:metadata-1-0/upnp/",
		Item: didlItem{
			ID:         "0",
			ParentID:   "-1",
			Restricted: "1",
			Title:      "Castor Stream",
			Class:      "object.item.videoItem",
			Res: didlRes{
				ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", entry.mime, entry.features),
				Value:        streamURL.String(),
			},
		},
	}

	data, err := xml.Marshal(item)
	if err != nil {
		return "", fmt.Errorf("marshaling DIDL-Lite: %w", err)
	}
	return string(data), nil
}
