# frozen_string_literal: true

# require 'debug'

require './config/extensions/array/wrap'
require './config/database'
require './app/contracts'
require './app/interactors'
require './app/routes'
require './app/serializers'
require './app/models'

run Application
