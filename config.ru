# frozen_string_literal: true

# require 'debug'

require './config/database'
require './app/contracts'
require './app/routes'
require './app/serializers'
require './app/models'

run Application
