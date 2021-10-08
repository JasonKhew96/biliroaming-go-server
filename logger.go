package main

import "go.uber.org/zap"

func initLogger(isDebug bool) (*zap.Logger, error) {
	if isDebug {
		return zap.NewDevelopment()
	} else {
		return zap.NewProduction()
	}
}
