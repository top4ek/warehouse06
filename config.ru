# config.ru
# frozen_string_literal: true

# ENV['RACK_ENV'] will be set to 'development' by default inside 'config/environment.rb' if not already set.
# Remove explicit ENV['RACK_ENV'] and Bundler/dotenv lines if they are now fully handled by environment.rb
require_relative './config/environment'

# Original lines specific to config.ru (after loading the app environment)
# Make sure RebuildDbService is loaded via config/environment.rb if it's still needed here.
# If RebuildDbService.new.call is for initial setup, consider if it should run every time the app starts,
# or only in specific environments (e.g., development).
# For now, keeping it as it was.
RebuildDbService.new.call

run Application
