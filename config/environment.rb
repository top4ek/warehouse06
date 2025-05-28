# config/environment.rb
# This file is responsible for loading all common application dependencies.

# Ensure Bundler is set up. If RACK_ENV is already set (e.g., by spec_helper), use it.
ENV['RACK_ENV'] ||= 'development' # Default for general use (e.g. rake tasks, console)

require 'bundler/setup'
Bundler.require(:default, ENV['RACK_ENV'].to_sym)
# Dotenv should be loaded based on RACK_ENV. If dotenv is in Gemfile,
# Bundler.require should handle its setup for automatic .env file loading.
# If explicit load is needed (e.g. if not using dotenv-rails's railtie):
# require 'dotenv/load' # This would load .env, .env.local, .env.<RACK_ENV>, .env.<RACK_ENV>.local

project_root = File.expand_path('..', __dir__) # Project root from this file's perspective (config/)

# Load application components
require File.join(project_root, 'config/extensions/array/wrap.rb')
require File.join(project_root, 'config/database.rb') # Connects to DB based on ENV['DATABASE_URL']
require File.join(project_root, 'app/models.rb')
require File.join(project_root, 'app/contracts.rb')
require File.join(project_root, 'app/services.rb')
require File.join(project_root, 'app/interactors.rb')
require File.join(project_root, 'app/serializers.rb')
require File.join(project_root, 'app/routes.rb') # Should define the 'Application' constant or main Roda app
