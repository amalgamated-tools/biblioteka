export interface BookAnnotation {
  id: string;
  user_id: string;
  book_id: string;
  text: string;
  cfi?: string;
  group_id?: string;
  user_name: string;
  created_at: string;
  updated_at: string;
}

export interface BookAnnotationInput {
  text: string;
  cfi?: string;
  group_id?: string;
}
