//TODO
// Do authoritative DNS lookup rather than just address?

package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	fGetConfig()
	if viper.GetString("debug") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debug("In main")
	fCompareAndRun()
}

func fCompareAndRun() {
	logrus.Debug("In function fCompareAndRun")
	currentIP := net.ParseIP(fGetCurrentIp())
	currentDNS := net.ParseIP(fGetCurrentDns())
	if currentIP == nil {
		logrus.Fatal("Current IP is not valid")
	}
	if currentDNS == nil {
		logrus.Fatal("Current DNS is not valid")
	}

	if string(currentIP) == string(currentDNS) {
		logrus.Info("ip matches dns, no change required")
		os.Exit(0)
	} else {
		logrus.Info("ip doesn't match dns - update wanted")
		fChangeIP(currentIP)
	}
}

func fChangeIP(requestedIP net.IP) {
	logrus.Debug("In function fChangeIP")
	resolver := viper.GetString("resolver")
	switch resolver {
	case "aws":
		fChangeAWS(requestedIP)
	case "nsone":
		fChangeNSONE()
	default:
		log.Fatal("resolver not set in config, or set to incorrect value")
	}
}

func fGetConfig() {
	logrus.Debug("In function fGetConfig")
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.AddConfigPath("/etc/ddns/")
	err := viper.ReadInConfig()
	if err != nil {
		logrus.Errorf("%v", err)
	}
}

func fGetCurrentIp() string {
	logrus.Debug("In fGetCurrentIP")
	var (
		finder = viper.GetStringSlice("ip_finder")
	)
	for _, v := range finder {
		resp, err := http.Get(v)
		if err != nil {
			logrus.Error("from %s: %v", v, err)
		} else {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("from %s: %v", v, err)
			}
			logrus.Info("Current IP address reported as: " + string(body))

			return (strings.TrimSpace(string(body)))

		}
	}
	logrus.Fatal("Failed to get IP")
	return "get IP failed"
}

func fGetCurrentDns() string {
	// need to do authoritative lookup here, avoid cache
	logrus.Debug("In function fGetCurrentDns")
	currentAddress, _ := net.LookupIP(viper.GetString("record"))
	for _, ip := range currentAddress {
		logrus.Info("current DNS reported as " + ip.String())
		return ip.String()
	}
	logrus.Fatal("fell out of the loop")
	return "DNS lookup failed"

}

func fChangeAWS(requestedIP net.IP) {
	logrus.Debug("In function fChangeAWS")

	sess, err := session.NewSession()
	if err != nil {
		log.Fatal("Failed to open AWS session")
	}
	svc := route53.New(sess)
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{ // Required
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: aws.String(viper.GetString("record")), // Required
						Type: aws.String("A"),                       // Required
						TTL:  aws.Int64(600),
						ResourceRecords: []*route53.ResourceRecord{
							{ // Required
								Value: aws.String(string(requestedIP)), // Required
							},
						},
					},
				},
			},
			Comment: aws.String("Changed by ddns script"),
		},
		HostedZoneId: aws.String(viper.GetString("aws_zone")), // Required
	}

	_, err = svc.ChangeResourceRecordSets(params)
	if err != nil {
		log.Fatalf("AWS Failed to update!:\n %v, %v", err)
	}
}

func fChangeNSONE() {
	logrus.Debug("This needs to be written!")
}
