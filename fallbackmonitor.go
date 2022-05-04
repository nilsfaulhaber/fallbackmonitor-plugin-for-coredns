package fallbackmonitor

import (
	"fmt"
	"context"
	"strings"
	"encoding/csv"
	"log"
	"os"
	"time"
"strconv"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metadata"
	"github.com/coredns/coredns/request"
	"util"
	"github.com/miekg/dns"
)

// Handler is a plugin handler that takes a query and templates a response.
type Handler struct {
	Zones []string
	Next      plugin.Handler
}


type queryData struct {
	Name     string
	Remote	 string
	Message  *dns.Msg
	md       map[string]metadata.Func
}

func (data *queryData) Meta(metaName string) string {
	if data.md == nil {
		return ""
	}

	if f, ok := data.md[metaName]; ok {
		return f()
	}

	return ""
}

// ServeDNS implements the plugin.Handler interface.
func (h Handler) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	protocol, _ := util.GetProtocolFromContext(ctx)
        fmt.Println("DNS Request over", protocol)

	data := getData(ctx, state)
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.Rcode = dns.RcodeSuccess

	rrs, err := assembleRR(data, protocol)
	if err != nil {
	   return dns.RcodeServerFailure, err
	}

	for i := range rrs {
	  msg.Answer = append(msg.Answer, rrs[i])
	}

	w.WriteMsg(msg)
	return dns.RcodeSuccess, nil

}

// Name implements the plugin.Handler interface.
func (h Handler) Name() string { return "fallbackmonitor" }

func assembleRR(data *queryData, protocol string) ([]dns.RR, error) {
	from_str := fmt.Sprintf("FROM_%s Protocol_%s", data.Remote, protocol)

	buffer_str := strings.Replace(data.Message.String(), "\n", "$", -1)
	buffer_str = strings.Replace(buffer_str, " ", "&", -1)
	buffer_str = strings.Replace(buffer_str, ";;", " ", -1)
	buffer_str = strings.Replace(buffer_str, ";", "%", -1)
	buffer_str = strings.Replace(buffer_str, "\t", "?", -1)
	buffer_str = strings.Replace(buffer_str, " &", " ", -1)
	buffer_str = strings.Replace(buffer_str, "& ", " ", -1)


	// ADD YOUR FILE PATH
	csvFile, err := os.OpenFile("/home/request_data.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)

	if err != nil {
		log.Fatalf("failed to open file file: %s", err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Comma = ';'
	content := fmt.Sprintf("%s %s", from_str, buffer_str)
	row := []string{data.Name, timestamp, content}
	erro := csvwriter.Write(row)
	if erro != nil {
	   fmt.Println(erro)
	   log.Fatalf("failed to write line: %s", erro)
	}

	csvwriter.Flush()
	if err := csvwriter.Error(); err != nil {
	  log.Fatal(err)
	}

	csvFile.Close()

	// 145 for 4KB responses, 72 for 2KB responses 
	rrs := make([]dns.RR, 145)
	// 145 for 4KB responses, 72 for 2KB responses
	for a := 0; a < 145; a++ {
	  base_str := fmt.Sprintf("%s IN AAAA 2003:ec:970e:f439:c5fd:30b8:2365:%x", data.Name, a)
	  rr, err := dns.NewRR(base_str)
          if err != nil {
             return rrs, err
          }
	 rrs[a] = rr
	}
	return rrs, nil
}

func getData(ctx context.Context, state request.Request) (*queryData) {
	data := &queryData{md: metadata.ValueFuncs(ctx), Remote: state.IP()}
	data.Name = state.Name()
        data.Message = state.Req
	return data
}
