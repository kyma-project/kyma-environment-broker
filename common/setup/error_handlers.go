package setup

import log "github.com/sirupsen/logrus"

func FatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func LogOnError(err error) {
	if err != nil {
		log.Error(err)
	}
}
