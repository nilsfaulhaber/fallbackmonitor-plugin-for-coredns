# fallbackmonitor-plugin-for-coredns
A CoreDNS plugin that returns large DNS responses, encodes received DNS queries and stores them in a local csv-file. The plugin was developed for research, to monitor the DoTCP fallback behavior of certain public DNS resolvers. The sender's IP address, the transport protocol (TCP/UDP) over that the request reached the name server and a timestamp are also stored. 

# Funcionality explained
For general details, how to develop custom CoreDNS plugins, please refer to [this article](https://coredns.io/2016/12/19/writing-plugins-for-coredns/).

## Retrieving the Transport Protocol
To retrieve the transport protocol the incoming DNS request was sent over, some small changes in the original CoreDNS code are necessary (in _core/dnsserver/server.go_). As a reference, the changed file was added to the repository and the changes are highlighted. 
To realize the retrieval of the transport protocol, the module _util.go_ needs to be added to the go source at first (usually located in _/usr/local/go/src_). server.go uses it to add the respective protocol to the context variable in its **Serve** and **ServePacket** function.

### Serve
~~~
s.server[tcp] = &dns.Server{Listener: l, Net: "tcp", MsgAcceptFunc: MyMsgAcceptFunc, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		...
		ctx = context.WithValue(ctx, util.CtxKey{}, "TCP")
		s.ServeDNS(ctx, w, r)
	})}

~~~

### ServePacket
~~~
s.server[udp] = &dns.Server{PacketConn: p, Net: "udp", MsgAcceptFunc: **MyMsgAcceptFunc**, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		...
		ctx = context.WithValue(ctx, util.CtxKey{}, "UDP")
		s.ServeDNS(ctx, w, r)
	})}
~~~
Note that we have noticed during tests that some requests using specific EDNS(0) Options were blocked by CoreDNS. Therefore, an own _MsgAcceptFunc_, **MyMsgAcceptFunc** was introduced to leave all request through to the plugin. In the **ServeDNS** (see fallbackmonitor.go) function, _fallbackmonitor_ firstly retrieves the transport protocol the request was sent over using the **context-variable (ctx)** passed as parameter: 

~~~
protocol, _ := util.GetProtocolFromContext(ctx)
~~~


## Retreiving Request Data 
The remaining data from the incoming request is as well taken from **ctx** in **ServeDNS**: 
~~~
data := getData(ctx, state)

msg := new(dns.Msg)
msg.SetReply(r)
msg.Authoritative = true
msg.Rcode = dns.RcodeSuccess

rr, err := assembleRR(data, protocol)
~~~

The body of the response message is filled by AAAA records returned from **assembleRR**.

**assbemleRR** takes the sender's IP (**data.Remote**) and the DNS message (**data.Message**), and encodes all the information.
~~~
func assembleRR(data *queryData, protocol string) (dns.RR, error) {
	from_str := fmt.Sprintf("FROM_%s Protocol_%s", data.Remote, protocol)

	buffer_str := strings.Replace(data.Message.String(), "\n", "$", -1)
	buffer_str = strings.Replace(buffer_str, " ", "&", -1)
	buffer_str = strings.Replace(buffer_str, ";;", " ", -1)
	buffer_str = strings.Replace(buffer_str, ";", "%", -1)
	buffer_str = strings.Replace(buffer_str, "\t", "?", -1)
	buffer_str = strings.Replace(buffer_str, " &", " ", -1)
	buffer_str = strings.Replace(buffer_str, "& ", " ", -1)
	str := fmt.Sprintf("%s IN TXT %s %s", data.Name, from_str, buffer_str)
~~~
Afterwards, a connection to a csv-file is established and the encoded message, a timestamp, and the domain that was requested for resolution are saved. 
~~~
	// ADD YOUR FILE PATH
	csvFile, err := os.OpenFile("/home/faulhabn/request_data.csv", os.O_APPEND|os.O_WRONLY, os.ModeAppend)

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
~~~
Finally, a specific number of different AAAA records is generated and returned. 
~~~
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
~~~

# Plugging it together
Follow this step by step guide to create a CoreDNS server running the fallbackmonitor plugin. 

1. Clone this repository 
2. Clone the [CoreDNS repository](https://github.com/coredns/coredns)
3. Add module **util.go** to go source 
Copy the module util.go to /usr/local/go/src/
4. Add mymsgacceptfunc.go to the coredns folder: _coredns/core/dnsserver/_
5. Replace the orginal file _server.go_ in _coredns/core/dnsserver/_ with the one from this repository or apply the highlighted changes to the original file
6. Create a folder _fallbackmonitor_ in _coredns/plugin/_.
7. Copy the files _fallbackmonitor.go_, _metrics.go_ and _setup.go_ the _coredns/plugin/fallbackmonitor_
8. Save the path of the csv-file to path that fits you and create an emtpy file with the respective name
9. Replace the file _coredns/plugin.cfg_ with the one in this repository or apply the highlighted changes
10. Build everything by running 
~~~~
make
~~~~~
in _coredns/_
The output is an executable file _coredns_ that can be deployed. 


Finally, create a Corefile similar to the one in this respository and place it into _coredns/_. 
~~~~
membrain-it.technology:53 {
    fallbackmonitor
}
~~~~
Make sure to replace _membrain-it.technology_ with the zone your name server is authoritative for. 

Follow one of the procedures listed [here](https://github.com/coredns/deployment) to deploy the server and to specify the configuration with the Corefile. We have chosen to deploy the service using **systemd** which worked perfectly fine. 
