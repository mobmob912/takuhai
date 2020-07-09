rm -f echo-amd64-$stepID
wget https://storage.googleapis.com/tamabus-binary/echo-amd64 -O echo-amd64-$stepID
chmod +x echo-amd64-$stepID
./echo-amd64-$stepID
