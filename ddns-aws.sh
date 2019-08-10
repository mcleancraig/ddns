#!/bin/bash

# Set yo shit up!!
APIKEY="7a4gElr2E9mIwnMwLbSj"
ZONE="fukka.co.uk" # Zone according to nsone
AWSZONE="Z159W85ZCGY57D" # Zone according to AWS
RECORD="fukka.co.uk" # Record you want to change within the zone
IP_FINDER_LIST="http://bot.whatismyipaddress.com" "http://ipinfo.io/ip"
resolver=aws

# To Do:
# Check current ip is valid, external IP - DONE
# check it isn't the currently registered IP
# Register it if needed

# Rename the functions to something more descriptive - DONE

#copied from http://www.linuxjournal.com/content/validating-ip-address-bash-script

function bail() {
 echo "Bailing out:$1"
 exit 1
 }

function debug() {
debug=true && echo "DEBUG: $1"
}

function is_valid_ip()
{
  local ip=$1
  local stat=1 # set to fail to danger

  if [[ $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
    OIFS=$IFS
    IFS='.'
    ip=($ip)
    IFS=$OIFS
    [[ ${ip[0]} -le 255 && ${ip[1]} -le 255 && ${ip[2]} -le 255 && ${ip[3]} -le 255 ]]
      stat=$?
  fi
  return $stat
}

#inspired by https://unix.stackexchange.com/questions/98923/programmatically-extract-private-ip-addresses
function is_public_ip()
{
  local ip=$1
  local stat=2 # should never fail with this code
  
  echo $ip | grep -q -E '^(192\.168|10\.|172\.1[6789]\.|172\.2[0-9]\.|172\.3[01]\.)'
  stat=$?
  return $stat
}

#Rewrite mode...

# Get my public IP
for address in ${IP_FINDER_LIST}
do
  debug "checking from ${finder}"
  IPADDR=$(wget -qO- ${finder})
  debud "returned ${IPADDR}"
  is_valid_ip ${IPADDR} && is_public_ip ${IPADDR} && break
done

# Do the do...
case resolver in 
 "nsone")
# NSOne version
curl -X POST -H "X-NSONE-Key: $APIKEY" -d '{
 "answers": [
  {
   "answer": [
    "'$IPADDR'"
   ]
  }
 ]
}' https://api.nsone.net/v1/zones/$ZONE/$RECORD/A ;;
 "aws")
# AWS version 
 aws route53 change-resource-record-sets --hosted-zone-id ${AWSZONE} --change-batch '{ "Comment": "Testing update of a record", "Changes": [ { "Action": "UPSERT", "ResourceRecordSet":{ "Name": "'$RECORD'", "Type": "A", "TTL": 100, "ResourceRecords": [ { "Value": "'$IPADDR'" } ] } } ] }' ;;
  *) bail "Invalid resolver specified: ${resolver}" ;;
esac

 
 
 
 
