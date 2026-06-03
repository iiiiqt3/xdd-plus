package Xml

import "encoding/xml"

type HongBao struct {
	XMLName xml.Name `xml:"msg"`
	Appmsg  struct {
		XMLName   xml.Name `xml:"appmsg"`
		Wcpayinfo struct {
			XMLName     xml.Name `xml:"wcpayinfo"`
			Templateid  string   `xml:"templateid,CDATA"`
			Nativeurl   string   `xml:"nativeurl,CDATA"`
			Sceneid     int      `xml:"sceneid,CDATA"`
			Paymsgid    string   `xml:"paymsgid,CDATA"`
			Invalidtime int64    `xml:"invalidtime,CDATA"`
		}
	}
}
