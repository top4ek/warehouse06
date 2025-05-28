# frozen_string_literal: true

source 'https://rubygems.org'

gem 'blueprinter'

gem 'puma'
gem 'roda'
gem 'slim'

gem 'sequel'
gem 'sqlite3'
gem 'zeitwerk' # Added as a direct dependency

gem 'front_matter_parser'
gem 'redcarpet'

gem 'dry-monads'
gem 'dry-validation'

gem 'debug'

group :development, :test do
  gem 'dotenv' # Was already in :development, :test
  gem 'guard'
  gem 'guard-rspec', require: false
  gem 'rerun' # Added rerun here
end

group :test do
  gem 'rspec'
  gem 'rack-test' # For testing Roda routes
  gem 'factory_bot'
end
