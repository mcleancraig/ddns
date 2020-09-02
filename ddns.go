package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	api "gopkg.in/ns1/ns1-go.v2/rest"
)

func main() {
	err := getConfig()
	if err != nil {
		log.Fatal(err)
	}
	if viper.GetString("debug") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	logrus.Debug("In main")
	err = compareAndRun()
	if err != nil {
		log.Fatal(err)
	} else {
		logrus.Info("Completed")
	}
}
func getConfig() (err error) {
	logrus.Debug("In function getConfig")
	viper.SetConfigName("ddns")
	viper.AddConfigPath("$HOME/.ddns/")
	viper.AddConfigPath("/etc/ddns/")
	err = viper.ReadInConfig()
	if err != nil {
		return err // not strictly needed as this is all it will return by default
	}
	// config checks

	return nil
}

func compareAndRun() (err error) {
	logrus.Debug("In function compareAndRun")
	currentIP, err := getCurrentIp()
	if err != nil {
		return errors.Errorf("Failed getting IP: %v", err)
	}
	currentDNS, err := getCurrentDns()
	if err != nil {
		return errors.Errorf("DNS Lookup failed: %v", err)
	}
	if currentIP == nil {
		return errors.New("Current IP is not valid")
	}
	if currentDNS == nil {
		return errors.New("Current DNS is not valid")
	}
	if string(currentDNS) == string(currentIP) {
		logrus.Info("IPs match - no action required")
		return nil
	}
	logrus.Info("ip doesn't match dns - update wanted")
	err = changeIP(currentIP)
	if err != nil {
		return
	} else {
		return nil
	}

}

func changeIP(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeIP")
	resolver := viper.GetString("resolver")
	switch resolver {
	case "aws":
		err = changeAWS(requestedIP)
		if err != nil {
			return
		}
	case "nsone":
		changeNSONE()
	default:
		logrus.Error("resolver not set in config, or set to incorrect value")
		return errors.New("resolver not set")
	}
	return nil
}

func getCurrentIp() (reportedIP net.IP, err error) {
	logrus.Debug("In function getCurrentIP")
	var (
		finder = viper.GetStringSlice("ip_finder")
	)
	for _, v := range finder {
		logrus.Debugf("Getting IP from %s", v)
		resp, err := http.Get(v)
		if err != nil {
			logrus.Errorf("from %s: %v", v, err)
		} else {
			defer resp.Body.Close()
			logrus.Debugf("Opening connection to %s", v)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("from %s: %v", v, err)
			}
			reportedIP := net.ParseIP(strings.TrimSpace(string(body)))
			logrus.Debugf("response from %v was: %v ", v, string(reportedIP))
			if reportedIP != nil {
				logrus.Infof("Current IP address reported by %v as: %v ", v, reportedIP)
				return reportedIP, nil

			} else {
				logrus.Errorf("Missed getting an IP from %s", v)
			}

		}
	}
	return net.ParseIP(""), errors.New("get IP failed")
}

func getCurrentDns() (_ net.IP, err error) {

	var outputIP net.IP
	targetname := viper.GetString("record")
	logrus.Debug("In function getCurrentDns")
	logrus.Debug("Finding nameservers for ", targetname)
	nameservers, err := net.LookupNS(targetname)
	if err != nil {
		return nil, err
	}
	logrus.Debug("Results: %v", nameservers)
	for _, ns := range nameservers {
		nshost := ns.Host
		logrus.Debugf("Looking up %v against %v", viper.GetString("record"), nshost)
		resolver := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, "udp", net.JoinHostPort(nshost, "53"))
			},
		}
		resp, err := resolver.LookupIPAddr(context.Background(), viper.GetString("record"))
		if err != nil {
			return nil, errors.Errorf("error from resolver: %v", err)
		}
		// Now we get an array back, so we need to find out what's in it
		for _, name := range resp {
			outputIP = net.ParseIP(name.String())
			logrus.Infof("DNS Lookup returned %s from %v", outputIP, nshost)
		}
		return outputIP, nil

	}
	return nil, errors.New("Fell out of nameserver loop without getting any results")

}

func changeAWS(requestedIP net.IP) (err error) {
	logrus.Debug("In function changeAWS")

	sess, err := session.NewSession()
	if err != nil {
		return errors.Errorf("Failed to open AWS session: %v", err)
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
								Value: aws.String(requestedIP.String()), // Required
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
		return errors.Errorf("AWS Failed to update!:\n %v", err)
	}
	return nil
}

func changeNSONE() {
	httpClient := &http.Client{Timeout: time.Second * 10}
	client := api.NewClient(httpClient, api.SetAPIKey(viper.GetString("api_key")))

	zones, _, err := client.Zones.List()
	if err != nil {
		log.Fatal(err)
	}

	for _, z := range zones {
		fmt.Println(z.Zone)
	}

}
