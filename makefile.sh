cd $HOME

docker image build -t axhb:v1.0 .
docker save -o axhb.img axhb:v1.0
docker run -itd --name axhb --rm -v /home/yuan/axhb/heartbeat.yml:/axhb/heartbeat.yml axhb:v1.0
