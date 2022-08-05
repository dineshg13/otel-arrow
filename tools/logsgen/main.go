/*
 * // Copyright The OpenTelemetry Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //       http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 *
 */

package main

import (
	"flag"
	"log"
	"os"
	"path"

	"google.golang.org/protobuf/proto"

	"otel-arrow-adapter/pkg/datagen"
)

var help = flag.Bool("help", false, "Show help")
var outputFile = "./data/otlp_logs.pb"
var batchSize = 1000

func main() {
	// Define the flags.
	flag.StringVar(&outputFile, "output", outputFile, "Output file")
	flag.IntVar(&batchSize, "batchsize", batchSize, "Batch size")

	// Parse the flag
	flag.Parse()

	// Usage Demo
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// Generate the dataset.
	generator := datagen.NewLogsGenerator(datagen.DefaultResourceAttributes(), datagen.DefaultInstrumentationScope())
	request := generator.Generate(batchSize, 100)

	// Marshal the request to bytes.
	msg, err := proto.Marshal(request)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	// Write protobuf to file
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		err = os.MkdirAll(path.Dir(outputFile), 0700)
		if err != nil {
			log.Fatal("error creating directory: ", err)
		}
	}
	err = os.WriteFile(outputFile, msg, 0644)
	if err != nil {
		log.Fatal("write error: ", err)
	}
}
