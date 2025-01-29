import { Component } from '@angular/core';
import { BookService } from './book.service';
import { Observable } from 'rxjs';

export interface WeatherData {
  current_weather: {
    temperature: number;
  };
}

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css'],
})
export class AppComponent {
  title = 'angular-rxjs';
}
