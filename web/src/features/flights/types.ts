
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

export interface FlightMeta {
  gufi: string
  callSign?: string
  registration?: string
  aircraftType?: string
  origin?: string
  destination?: string
  status?: string
  actualDepartureTime?: string
  actualArrivalTime?: string
  dropCount: number
  reactivationCount: number
  firstSeen: string
  lastSeen: string
}

export interface TrackPoint {
  time: string
  lat?: number
  lon?: number
  alt?: number
  heading?: number
  speedKt?: number
  status?: string
}

export interface FlightRecord {
  flight: FlightMeta
  track: TrackPoint[]
}
