# Guardfile
# More info at https://github.com/guard/guard#readme

# RSpec plugin
# More info at https://github.com/guard/guard-rspec#readme
guard :rspec, cmd: "bundle exec rspec" do
  # Standard RSpec specs - run all specs if any spec file changes or is added/removed
  # This is a common behavior; alternatively, could run only the changed spec.
  # For simplicity, let's stick to a simple pattern that re-runs the specific changed spec.
  watch(%r{^spec/.+_spec\.rb$})

  # Watch application files and attempt to run corresponding specs
  # This tries common conventions. If a matching spec isn't found, Guard handles it.
  
  # Watch files directly under app/ (e.g., app/models.rb, app/routes.rb)
  watch(%r{^app/([^/]+)\.rb$}) do |m|
    [
      "spec/app/#{m[1]}_spec.rb",      # e.g. app/services.rb -> spec/app/services_spec.rb
      "spec/models/#{m[1]}_spec.rb",  # e.g. app/models.rb -> spec/models/models_spec.rb (less common)
      # Add more specific mappings if your project has top-level app files with direct specs
    ]
  end

  # Watch files in subdirectories of app/ (e.g., app/models/node.rb)
  watch(%r{^app/(\w+)/(.+)\.rb$}) do |m|
    # m[1] is the subdirectory (e.g., "models", "services")
    # m[2] is the file name without .rb (e.g., "node")
    [
      "spec/#{m[1]}/#{m[2]}_spec.rb",              # General mapping: app/models/foo.rb -> spec/models/foo_spec.rb
      "spec/requests/#{m[1]}/#{m[2]}_spec.rb"    # For routes: app/routes/api.rb -> spec/requests/routes/api_spec.rb (adjust as needed)
    ]
  end
  
  # If you have a lib directory
  # watch(%r{^lib/(.+)\.rb$})     { |m| "spec/lib/#{m[1]}_spec.rb" }

  # Watch spec helper and main config files - run all specs if these change
  watch('spec/spec_helper.rb')  { "spec" }
  watch('config.ru') { "spec" }
  # Consider adding other key config files that would affect all tests
  # watch('config/database.rb') { "spec" }

  # Remove the old dsl lines:
  # require "guard/rspec/dsl"
  # dsl = Guard::RSpec::Dsl.new(self)
  # watch(dsl.specs)
  # watch(dsl.app_files) { |m| dsl.run_matching_specs(m) }
end

# Example for RuboCop (can be uncommented if RuboCop is added)
# guard :rubocop do
#   watch(/.+\.rb$/)
#   watch(/.+\.rake$/)
#   watch(%r{(?:Guardfile|Gemfile|Rakefile|\.ru|\.gemspec)$})
# end
