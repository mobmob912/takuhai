rm -f store-log-$stepID
wget https://storage.googleapis.com/tamabus-binary/store-log -O store-log-$stepID
chmod +x store-log-$stepID
./store-log-$stepID
