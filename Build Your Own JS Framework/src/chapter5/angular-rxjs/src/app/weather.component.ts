import { Component, OnInit } from '@angular/core';
import { BookService } from './book.service';
import { catchError } from 'rxjs/operators';
import { of } from 'rxjs';

@Component({
  selector: 'app-weather',
  styleUrls: ['./weather.component.css'],

  template: `
    <ng-template #loading>Loading...</ng-template>
    <div class="card-container">
      <div class="card">
        <svg
          class="material-icons"
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
        >
          <path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z" />
        </svg>
        <span
          >Temperature:
          <strong
            >{{ weatherData?.current_weather?.temperature }} °C</strong
          ></span
        >
      </div>

      <div class="card">
        <svg
          class="material-icons"
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          viewBox="0 0 24 24"
        >
          <path d="M10 6L8.59 7.41 13.17 12l-4.58 4.59L10 18l6-6z" />
        </svg>
        <span
          >Wind Speed:
          <strong>{{ weatherData?.current_weather?.windspeed }}</strong></span
        >
      </div>
    </div>
    <div *ngIf="weatherData; else loading">
      <h1>Weather Data</h1>
      <pre>{{ weatherData | json }}</pre>
    </div>
  `,
})
export class WeatherComponent implements OnInit {
  weatherData: any;

  constructor(private weatherService: BookService) {}

  ngOnInit(): void {
    this.weatherService
      .getWeather(43.6532, 79.3832)
      .pipe(
        catchError((err) => {
          console.error('An error occurred', err);
          return of(null);
        })
      )
      .subscribe((data) => {
        console.log('Weather Data:', data);
        this.weatherData = data;
      });
  }
}
