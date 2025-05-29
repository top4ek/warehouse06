# frozen_string_literal: true

ENV['RACK_ENV'] ||= 'development'

require 'bundler/setup'
Bundler.require(:default, ENV['RACK_ENV'].to_sym)

require 'dotenv/load'

project_root = File.expand_path('..', __dir__)

require File.join(project_root, 'config/extensions/array/wrap.rb')
require File.join(project_root, 'config/database.rb')

require 'zeitwerk'
loader = Zeitwerk::Loader.new

loader.push_dir(File.join(project_root, 'app'))

app_subdirs_to_collapse = %w[
  models
  services
  interactors
]
app_subdirs_to_collapse.each do |subdir|
  dir_path = File.join(project_root, 'app', subdir)
  loader.collapse(dir_path) if Dir.exist?(dir_path)
end

loader.enable_reloading if ENV['RACK_ENV'] == 'development'

loader.setup
loader.reload
