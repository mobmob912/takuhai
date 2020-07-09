rm -f receive-temperature-$stepID
wget https://storage.googleapis.com/tamabus-binary/receive-temperature-amd64 -O receive-temperature-$stepID
chmod +x receive-temperature-$stepID
./receive-temperature-$stepID
