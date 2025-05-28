# config/environment.rb
ENV['RACK_ENV'] ||= 'development' # Default for general use

require 'bundler/setup'
Bundler.require(:default, ENV['RACK_ENV'].to_sym)

# Ensure .env files are loaded. This is crucial before any app code relies on ENV vars.
# If RACK_ENV is 'test' (set by spec_helper), this will load .env.test.
require 'dotenv/load'

# Define project_root as the directory containing this file (config directory).
project_root = File.expand_path('..', __dir__) # Corrected definition as per instruction

# --- Files that need to be loaded before Zeitwerk ---
# Extensions, initializers, or configurations that don't follow Zeitwerk's conventions
# or need to be available very early.
# These paths should be relative to the project_root or absolute.
require File.join(project_root, 'config/extensions/array/wrap.rb')
require File.join(project_root, 'config/database.rb') # Establishes DB connection

# --- Zeitwerk Setup ---
require 'zeitwerk'
loader = Zeitwerk::Loader.new

# Add directories to be managed by Zeitwerk.
# Use project_root directly.
loader.push_dir(File.join(project_root, 'app'))
# If you had a 'lib' directory you wanted Zeitwerk to manage:
# loader.push_dir(File.join(project_root, 'lib'))

# Collapse specific directories under 'app' if their structure doesn't map to namespaces.
app_subdirs_to_collapse = %w[models contracts services interactors serializers routes]
app_subdirs_to_collapse.each do |subdir|
  dir_path = File.join(project_root, 'app', subdir) # Use project_root
  loader.collapse(dir_path) if Dir.exist?(dir_path)
end

# Enable reloading with Zeitwerk if you want to use its reloading capabilities
if ENV['RACK_ENV'] == 'development'
  loader.enable_reloading
end

loader.setup # Finalize Zeitwerk setup. Constants are now autoloadable.

# --- Files/code to run AFTER Zeitwerk setup (if any) ---
# For example, if some code needs all constants to be loaded.

# The previous explicit requires for files under 'app/' are handled by Zeitwerk.
