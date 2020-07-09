rm -f echo-$stepID
wget https://storage.googleapis.com/tamabus-binary/echo -O echo-$stepID
chmod +x echo-$stepID
./echo-$stepID
