Official docs confirm the overall shape: Alexa custom skills can use AWS Lambda as the backend, Lambda must be connected to the skill
  endpoint by ARN, and Alexa should be added as a Lambda trigger with Skill ID verification. Amazon also documents the JSON editor and
  “Build skill” flow for the interaction model. Sources: Alexa Lambda hosting docs, Lambda env var docs, Lambda zip docs, Alexa
  interaction model docs. (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/host-a-custom-skill-as-an-aws-lambda-function.html)) (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/host-a-custom-skill-as-an-aws-lambda-function.html)) (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/host-a-custom-skill-as-an-aws-lambda-function.html)) (docs.aws.amazon.com
  (https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html)) (docs.aws.amazon.com
  (https://docs.aws.amazon.com/lambda/latest/dg/nodejs-package.html)) (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/create-the-interaction-model-for-your-skill.html)) (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/create-the-interaction-model-for-your-skill.html))

  0. Prereqs
  You need:

  - Amazon Developer account: https://developer.amazon.com/alexa/console/ask
  - AWS account: https://aws.amazon.com/
  - Your public Cloudflare URL for revinder_bridge
  - Your bridge token: same value used as HOME_TASKS_TOKEN

  Also confirm this works from outside your LAN:

  curl -i https://your-cloudflare-host.example/health

  1. Package Lambda
  From your machine:

  cd /Users/seanottey/projects/revinder/revinder_alexa_skill/lambda
  zip -r ../lambda.zip index.js package.json

  No npm install is needed right now because the Lambda code has no third-party dependencies.

  2. Create AWS Lambda
  In AWS Console:

  1. Open AWS Lambda.
  2. Pick region us-east-1 for US English.
  3. Click Create function.
  4. Choose Author from scratch.
  5. Function name:

  revinder-alexa-skill

  6. Runtime: Node.js 20+ or latest available Node.js.
  7. Architecture: x86_64.
  8. Permissions: choose the default “create a new role with basic Lambda permissions.”
  9. Create function.

  AWS’s own Lambda walkthrough uses the same basic console flow: create function, author from scratch, choose runtime, create function.
  (docs.aws.amazon.com (https://docs.aws.amazon.com/lambda/latest/dg/getting-started.html))

  3. Upload Code
  In the Lambda function:

  1. Go to Code tab.
  2. Choose Upload from / .zip file.
  3. Upload:

  /Users/seanottey/projects/revinder/revinder_alexa_skill/lambda.zip

  4. Confirm handler is:

  index.handler

  The zip must have index.js at the zip root, which is how the current package command builds it. AWS documents that Node zip deployments
  need the handler file at the root. (docs.aws.amazon.com (https://docs.aws.amazon.com/lambda/latest/dg/nodejs-package.html))

  4. Set Lambda Environment Variables
  In Lambda:

  1. Go to Configuration.
  2. Open Environment variables.
  3. Add:

  REVINDER_BRIDGE_BASE_URL=https://your-cloudflare-host.example
  REVINDER_BRIDGE_TOKEN=your-home-tasks-token
  DEFAULT_TIME_ZONE=America/Los_Angeles

  4. Save.

  AWS documents Lambda env vars as key/value config available to code through process.env. (docs.aws.amazon.com
  (https://docs.aws.amazon.com/lambda/latest/dg/configuration-envvars.html))

  5. Create Alexa Skill
  In Alexa Developer Console:

  1. Go to https://developer.amazon.com/alexa/console/ask
  2. Click Create Skill.
  3. Skill name:

  revinder

  4. Primary locale: English (US).
  5. Experience/model: Other or Custom depending on console wording.
  6. Backend/resources: choose Provision your own or equivalent, because we are using your AWS Lambda.
  7. Create skill.

  6. Import Interaction Model
  In the Alexa skill:

  1. Go to Build.
  2. Go to Custom → JSON Editor.
  3. Paste the contents of:

  /Users/seanottey/projects/revinder/revinder_alexa_skill/skill-package/interactionModels/custom/en-US.json

  4. Save.
  5. Click Build Skill / Build Model.

  Amazon documents editing the interaction model through Custom > JSON Editor, then saving and building the model. (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/create-the-interaction-model-for-your-skill.html)) (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/create-the-interaction-model-for-your-skill.html))

  7. Connect Skill to Lambda
  In AWS Lambda:

  1. Copy the Lambda ARN from the function page.

  In Alexa Developer Console:

  1. Go to Build → Endpoint.
  2. Service endpoint type: AWS Lambda ARN.
  3. Paste the Lambda ARN into Default Region.
  4. Save.

  Amazon’s docs say to copy the Lambda ARN and paste it into the skill’s Custom > Endpoint Lambda ARN field. (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/host-a-custom-skill-as-an-aws-lambda-function.html))

  8. Add Alexa Trigger to Lambda
  In Alexa Developer Console:

  1. Go back to your skill list.
  2. Copy the Skill ID.

  In AWS Lambda:

  1. Open revinder-alexa-skill.
  2. Go to Configuration.
  3. Add trigger.
  4. Select Alexa Skills Kit.
  5. Enable Skill ID verification.
  6. Paste the Skill ID.
  7. Save.

  Amazon recommends Skill ID verification so only your skill can invoke the Lambda. (developer.amazon.com
  (https://developer.amazon.com/en-US/docs/alexa/custom-skills/host-a-custom-skill-as-an-aws-lambda-function.html))

  9. Test
  In Alexa Developer Console:

  1. Go to Test tab.
  2. Enable testing for development.
  3. Type or say:

  ask revinder to add a task on Tuesday at 8pm do that one thing with tags home and cottage

  Expected Alexa response:

  Added.

  Then verify bridge:

  curl -s https://your-cloudflare-host.example/api/tasks/pending \
    -H "Authorization: Bearer your-home-tasks-token"

  You should see a task with:

  {
    "title": "do that one thing",
    "source": "alexa",
    "tags": ["home", "cottage"]
  }

  One thing to watch: Alexa model builds can be picky about free-form slots. If the model build rejects AMAZON.SearchQuery usage, send me
  the exact error and I’ll adjust the interaction model.