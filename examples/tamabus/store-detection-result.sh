rm -f store-detection-result-$stepID
wget https://storage.googleapis.com/tamabus-binary/store-congestion-result -O store-detection-result-$stepID
chmod +x store-detection-result-$stepID
./store-detection-result-$stepID
