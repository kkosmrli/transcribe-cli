# Transcribe CLI

Transcribe CLI is a command line interface for transcribing audio files using OpenAI's Speech-to-Text API and GPT-4omini model.

# Usage

Set your OpenAI API key as an environment variable either via `export API_KEY=<your_key>` or by creating a `.env` file in the root directory of the project with the same variable.

````
go build
./transcribe -input=<path-to-audio-file> -compress 
````

> Note: The `-compress` flag is optional and is used to compress the audio file before transcribing it. For this to work, you need to have ffmpeg installed and set via $PATH on your system.

In order to receive the best results adjust the transcription and system prompt based on your needs. The default prompts are working for a german interview between two people.