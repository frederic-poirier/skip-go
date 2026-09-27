import {
    ClientMessageTypeGameDiscard,
    ClientMessageTypeGamePlayFromDiscard,
    ClientMessageTypeGamePlayFromHand,
    ClientMessageTypeGamePlayFromStock,
    ClientMessageTypeLobbyStart,
    ErrorPayload,
    ServerMessageTypeError,
    ServerMessageTypeGameState,
    ServerMessageTypeRoomState,
    type DiscardPayload,
    type GameView,
    type PlayFromDiscardPayload,
    type PlayFromHandPayload,
    type PlayFromStockPayload,
    type RoomView,
} from "./generated";

export type ClientMessage =
    | { type: typeof ClientMessageTypeGameDiscard; payload: DiscardPayload }
    | { type: typeof ClientMessageTypeGamePlayFromDiscard; payload: PlayFromDiscardPayload }
    | { type: typeof ClientMessageTypeGamePlayFromHand; payload: PlayFromHandPayload }
    | { type: typeof ClientMessageTypeGamePlayFromStock; payload: PlayFromStockPayload }
    | { type: typeof ClientMessageTypeLobbyStart };

export type ServerMessage =
    | { type: typeof ServerMessageTypeGameState; payload: GameView }
    | { type: typeof ServerMessageTypeRoomState; payload: RoomView }
    | { type: typeof ServerMessageTypeError; payload: ErrorPayload };
