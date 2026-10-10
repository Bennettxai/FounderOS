export type BlockDragOverPosition = 'before' | 'after';

export interface BlockDragState {
	isDragging: boolean;
	isDragOver: boolean;
	dragOverPosition: BlockDragOverPosition | null;
}

export interface BlockDragRect {
	top: number;
	height: number;
}

export interface BlockDropMoveInput {
	fromIndex: number;
	targetIndex: number;
	position: BlockDragOverPosition;
}

export interface BlockMove {
	fromIndex: number;
	toIndex: number;
}

export interface WriteBlockDragDataInput {
	index: number;
	blockId: string;
}

export const BLOCK_DRAG_INDEX_MIME = 'text/plain';
export const BLOCK_DRAG_ID_MIME = 'application/x-block-id';

export function createBlockDragState(overrides: Partial<BlockDragState> = {}): BlockDragState {
	return {
		isDragging: false,
		isDragOver: false,
		dragOverPosition: null,
		...overrides
	};
}

export function startBlockDrag(state: BlockDragState): BlockDragState {
	return {
		...state,
		isDragging: true
	};
}

export function clearBlockDragState(state: BlockDragState): BlockDragState {
	return {
		...state,
		isDragging: false,
		isDragOver: false,
		dragOverPosition: null
	};
}

export function setBlockDragOver(
	state: BlockDragState,
	position: BlockDragOverPosition
): BlockDragState {
	return {
		...state,
		isDragOver: true,
		dragOverPosition: position
	};
}

export function clearBlockDragOver(state: BlockDragState): BlockDragState {
	return {
		...state,
		isDragOver: false,
		dragOverPosition: null
	};
}

export function getBlockDragOverPosition(clientY: number, rect: BlockDragRect): BlockDragOverPosition {
	const midpoint = rect.top + rect.height / 2;
	return clientY < midpoint ? 'before' : 'after';
}

export function getBlockDropMove({
	fromIndex,
	targetIndex,
	position
}: BlockDropMoveInput): BlockMove | null {
	if (!Number.isInteger(fromIndex) || !Number.isInteger(targetIndex)) {
		return null;
	}

	const insertionIndex = position === 'after' ? targetIndex + 1 : targetIndex;

	if (fromIndex === insertionIndex || fromIndex + 1 === insertionIndex) {
		return null;
	}

	return {
		fromIndex,
		toIndex: fromIndex < insertionIndex ? insertionIndex - 1 : insertionIndex
	};
}

export function parseBlockDragIndex(value: string): number | null {
	const index = Number.parseInt(value, 10);
	return Number.isNaN(index) ? null : index;
}

export function writeBlockDragData(
	dataTransfer: Pick<DataTransfer, 'effectAllowed' | 'setData'>,
	{ index, blockId }: WriteBlockDragDataInput
): void {
	dataTransfer.effectAllowed = 'move';
	dataTransfer.setData(BLOCK_DRAG_INDEX_MIME, index.toString());
	dataTransfer.setData(BLOCK_DRAG_ID_MIME, blockId);
}

export function readBlockDragIndex(dataTransfer: Pick<DataTransfer, 'getData'>): number | null {
	return parseBlockDragIndex(dataTransfer.getData(BLOCK_DRAG_INDEX_MIME));
}

export function allowBlockDrop(dataTransfer: Pick<DataTransfer, 'dropEffect'>): void {
	dataTransfer.dropEffect = 'move';
}
