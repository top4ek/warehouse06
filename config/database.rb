# frozen_string_literal: true

require 'sequel'
require 'logger'

Database = Sequel.sqlite(ENV['DATABASE_NAME'] || ':memory:', logger: Logger.new($stderr))

Sequel.extension :migration
Sequel::Migrator.run(Database, 'config/migrations', use_transactions: true)
