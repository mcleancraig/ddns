package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/spf13/viper"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"
)

func f_debug(msg string) {
	if viper.Get("debug") == true {
		caller, _, _, _ := runtime.Caller(1)
		cname := runtime.FuncForPC(caller).Name()
		fmt.Println("DEBUG FROM: " + cname + " : " + msg)
	}
}
func bail(msg error) {
	f_debug("In function")
	panic(fmt.Errorf("Fatal error : %s \n", msg))
}

func main() {
	fGetConfig()
	f_debug("In function")
	//currentDns := fGetCurrentDns()
	//f_debug("Current DNS passed back is: " + fGetCurrentDns())
	//currentIP: = fGetCurrentIp()
	//f_debug("current reported IP is: " + fGetCurrentIp())
	//fChangeIfNeeded()

	if fGetCurrentDns() == fGetCurrentIp() {
		f_debug("IPs match - not changing record")
	} else {
		resolver := viper.GetString("resolver")
		switch resolver {
		case "aws":
			sess, err := session.NewSession()
			if err != nil {
				fmt.Println("failed to create session,", err)
				return
			}
			svc := route53.New(sess)
			fChangeAWS(svc)
		case "nsone":
			fChangeNSONE()
		default:
			f_debug("resolver not set in config, or set to incorrect value")
		}
	}
}

func fGetConfig() {
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.SetDefault("debug", true)
	f_debug("In function")
	err := viper.ReadInConfig()
	if err != nil {
		bail(err)
	}
}

func fGetCurrentIp() string {
	// need to spot bullshit responses from here...
	f_debug("In function")
	var (
		//body   string
		finder = viper.GetStringSlice("ip_finder")
	)
	fmt.Printf("Type: %T,  Size: %d \n", finder, len(finder))
	for i, v := range finder {
		fmt.Printf("Index: %d, Value: %v\n", i, v)
		resp, err := http.Get(v)
		if err != nil {
			bail(err)
		} else {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				bail(err)
			}
			_ = resp.Body.Close()
			f_debug("Current IP address reported as: " + string(body))
			return string(body)

		}
	}
	return "Failed to get IP"
}

func fGetCurrentDns() string {
	// need to do authoritative lookup here, avoid cache
	f_debug("In function")
	current_dns, _ := net.LookupHost(viper.GetString("record"))
	f_debug("Current DNS entry: " + strings.Join(current_dns, "."))
	return strings.Join(current_dns, ".")

}

func fChangeAWS(svc *route53.Route53) {

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{ // Required
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: aws.String(viper.GetString("record")), // Required
						Type: aws.String("A"),                       // Required
						ResourceRecords: []*route53.ResourceRecord{
							{ // Required
								Value: aws.String(viper.GetString("record")), // Required
							},
						},
						TTL:           aws.Int64(600),
						Weight:        aws.Int64(1),
						SetIdentifier: aws.String("Changed by ddns"),
					},
				},
			},
			Comment: aws.String("Changed by ddns script"),
		},
		HostedZoneId: aws.String(viper.GetString("aws_zone")), // Required
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	fmt.Printf("%v: %v\n", err, resp)
}

func fChangeNSONE() {

}
