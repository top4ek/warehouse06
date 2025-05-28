# frozen_string_literal: true

require 'bundler/setup'
ENV['RACK_ENV'] ||= 'development'
Bundler.require(:default, ENV['RACK_ENV'].to_sym)
require 'dotenv/load'

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
