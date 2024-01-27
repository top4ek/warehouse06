# frozen_string_literal: true

require 'sequel'

Database = Sequel.postgres(host: ENV['DATABASE_ADDRESS'],
                           user: ENV['DATABASE_USERNAME'],
                           password: ENV['DATABASE_PASSWORD'],
                           database: ENV['DATABASE_NAME'])

Sequel.extension :migration
Sequel::Migrator.run(Database, 'config/migrations')
