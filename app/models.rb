# frozen_string_literal: true

Sequel::Model.default_set_fields_options[:missing] = :skip
# Sequel::Model.plugin :timestamps

# require_relative 'models/tag' # Removed by Zeitwerk integration
# require_relative 'models/tagging' # Removed by Zeitwerk integration
# require_relative 'models/node' # Removed by Zeitwerk integration
