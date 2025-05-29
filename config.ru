# config.ru
# frozen_string_literal: true

require_relative './config/environment'

# RebuildDbService.new.call

run Routes::Base
