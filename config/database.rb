# frozen_string_literal: true

require 'sequel'
require 'logger'

logger = ENV['RACK_ENV'] == 'development' ? Logger.new($stderr) : nil

Database = Sequel.sqlite(ENV['DATABASE_NAME'] || ':memory:', logger:)

Sequel.extension :migration
Sequel::Migrator.run(Database, 'config/migrations', use_transactions: true)
