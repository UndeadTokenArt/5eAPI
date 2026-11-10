Here is the command to test the server locally

cd /home/undeadtoken/workspace/5eAPI

mkdir -p /tmp/5eapi-local-test

CHARACTER_GUEST_ACCESS=true \
CHARACTER_DATA_FILE=/tmp/5eapi-local-test/characters.json \
CAMPAIGN_DATA_FILE=/tmp/5eapi-local-test/campaigns.json \
go run .