import { Injectable } from '@nestjs/common';

@Injectable()
export class AppService {
  getShowcase(): string {
    return 'This is an example of a response from the Nest.js app for the framework showcase!';
  }
}
