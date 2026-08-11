
export interface Flight {
  gufi: string
  callSign: string
  registration?: string
  status: string
  lat: number
  lon: number
  alt?: number
  altUom?: string
  heading?: number
  speedKt?: number
  positionTime?: string
  timestamp?: string
  updatedAt?: string
}

export interface FlightsResponse {
  count: number
  flights: Flight[]
}
