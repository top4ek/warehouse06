# spec/spec_helper.rb
ENV['RACK_ENV'] = 'test' # CRITICAL: Set RACK_ENV to 'test' BEFORE loading environment

# If .env files are not loaded automatically by Bundler/dotenv setup for test env,
# ensure .env.test is loaded. 'dotenv/load' handles this well.
# This needs to happen after RACK_ENV is set to 'test' and before 'config/environment'
# might try to load a different .env file based on a default RACK_ENV.
require 'dotenv/load'

# Load the application using the new central environment file
require_relative '../config/environment'

require 'rack/test'
require 'factory_bot'

RSpec.configure do |config|
  config.include Rack::Test::Methods
  config.include FactoryBot::Syntax::Methods

  config.before(:suite) do
    FactoryBot.find_definitions
  end

  # config.around(:each) do |example|
  #   DB.transaction(rollback: :always, auto_savepoint: true) { example.run }
  # end

  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups
  config.filter_run_when_matching :focus
  config.example_status_persistence_file_path = "spec/examples.txt"
  config.disable_monkey_patching!
  config.warnings = true

  if config.files_to_run.one?
    config.default_formatter = "doc"
  end

  config.order = :random
  Kernel.srand config.seed
end

def app
  Application # Assuming 'Application' is defined after loading 'config/environment.rb'
end
