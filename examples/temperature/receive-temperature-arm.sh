rm -f receive-temperature-$stepID
wget https://storage.googleapis.com/tamabus-binary/receive-temperature-arm -O receive-temperature-$stepID
chmod +x receive-temperature-$stepID
./receive-temperature-$stepID
