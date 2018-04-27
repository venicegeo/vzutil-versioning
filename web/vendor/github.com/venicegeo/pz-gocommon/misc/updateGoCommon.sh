cd $1
apiEndpoint=https://api.github.com
getCommitCurl=`curl "$apiEndpoint/repos/venicegeo/pz-gocommon/commits" --write-out %{http_code} 2>/dev/null`
http_code=${getCommitCurl: -3}
if [ "$http_code" -ne 200 ]; then
  echo "Unable to get commit sha"
  exit 1
fi
regex='^\[ { "sha": "([^"]+)'
sha=`echo $getCommitCurl | grep -Eo "$regex" | cut -d\" -f4-`
echo "Sha:"
echo $sha
toReplace=`sed -n '/name: github.com\/venicegeo\/pz-gocommon/{N;p}' glide.lock`
#echo "$toReplace"
replacement=$'- name: github.com/venicegeo/pz-gocommon\n  version: '$sha
#echo "$replacement"
dat=`cat glide.lock`
result=`echo "${dat/"$toReplace"/"$replacement"}"`
#echo "$result"
echo "$result" > glide.lock
glide install -v
