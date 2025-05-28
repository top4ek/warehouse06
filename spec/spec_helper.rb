# spec/spec_helper.rb
ENV['RACK_ENV'] = 'test' # Ensure all tests run in the test environment

require 'bundler/setup' # Sets up load paths from Gemfile
# No Bundler.require here, RSpec will require gems as needed or they are required by app

require 'rack/test' # For testing Roda applications

# Adjust the path as necessary to load your application's main file or config.ru
# This assumes your application can be loaded by requiring config.ru
# or a specific app file. For Roda, often it's the file defining the App class.
# Since config.ru loads everything, let's try that.
require File.expand_path('../config.ru', __dir__)

require 'factory_bot' # Added for FactoryBot

RSpec.configure do |config|
  config.include Rack::Test::Methods # Provides methods like get, post, last_response

  # FactoryBot configuration
  config.include FactoryBot::Syntax::Methods

  config.before(:suite) do
    FactoryBot.find_definitions # Auto-discover factories
  end

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

  # config.profile_examples = 10 # Uncomment to show slow examples
  config.order = :random
  Kernel.srand config.seed
  
  # Add any other RSpec configurations here.
  # For example, database cleaning strategies if you were using a persistent test DB.
  # config.before(:suite) do
  #   # Setup tasks, e.g., DB cleaning
  # end
  # config.around(:each) do |example|
  #   # DB transaction wrapping
  #   DB.transaction(rollback: :always, auto_savepoint: true) { example.run }
  # end
end

# Helper method to get the Roda application instance for Rack::Test
# This might need to be adjusted based on how Application is defined and run in config.ru
def app
  Application # Assuming 'Application' is the class name of your Roda app
end
