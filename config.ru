# frozen_string_literal: true

require 'debug'

require 'dry/monads'

require './config/extensions/array/wrap'
require './config/database'
require './app/contracts'
require './app/services'
require './app/interactors'
require './app/routes'
require './app/serializers'
require './app/models'

RebuildDbService.new.call

run Application
