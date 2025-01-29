// Importing essential Angular and RxJS modules
import { Injectable } from '@angular/core'; // Injectable decorator to make this class injectable
import { HttpClient } from '@angular/common/http'; // HttpClient to make API requests
import { Observable, retry } from 'rxjs'; // Observable from RxJS to handle asynchronous operations
// Injectable decorator to make this service class injectable throughout the Angular application
@Injectable({
  providedIn: 'root',
})
export class BookService {
  private baseUrl = 'https://api.open-meteo.com/v1/forecast';

  constructor(private httpClient: HttpClient) {}

  // Method to fetch weather data based on latitude and longitude
  // It returns an Observable that will contain the API response
  getWeather(latitude: number, longitude: number): Observable<any> {
    const url = `${this.baseUrl}?latitude=${latitude}&longitude=${longitude}&current_weather=true`;
    // Using the retry operator in the pipeline to retry the request 3 times in case of failure
    return this.httpClient.get(url).pipe(retry(3));
  }
}
