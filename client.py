import asyncio
import sys

from viam.robot.client import RobotClient
from viam.rpc.dial import Credentials, DialOptions
from viam.components.camera import Camera
from viam.services.vision import VisionClient



async def connect():
    opts = RobotClient.Options.with_api_key(
      api_key='<API-KEY>',
      api_key_id='<API-KEY-ID>'
    )
    return await RobotClient.at_address('<YOUR-SMART-MACHINE-URL>', opts)

async def main():
    robot = await connect()
    # objectfilter client
    objectfilter = Camera.from_robot(robot, "objectfilter")
    objectfilter_return_value = await objectfilter.do_command({"vision-service":sys.argv[1]})
    print(f"Do_Command result: {objectfilter_return_value}")
  
    # Don't forget to close the machine when you're done!
    await robot.close()

if __name__ == '__main__':
    asyncio.run(main())
