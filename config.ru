# frozen_string_literal: true

require 'bundler/setup' # Added
ENV['RACK_ENV'] ||= 'development' # Added - Default RACK_ENV
Bundler.require(:default, ENV['RACK_ENV'].to_sym) # Added - Load gems
require 'dotenv/load' # Load .env files after gems are required

require 'debug' # This was originally here, keep it after Bundler.require

require 'dry/monads' # This was originally here

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
