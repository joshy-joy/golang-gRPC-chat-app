package main

import "errors"

var channelAlreadyExistError = errors.New("channel already exist. Try another channel name")
var channelNotFoundError = errors.New("channel not found")
var userNotFoundExistError = errors.New("user not connected to the server")
var userNotActiveError = errors.New("user not active")
