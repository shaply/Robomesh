- Robot card
- Login and logout
- Make the refresh robot cards not reload all of them and rather add new ones to the end of the list and update information on existing ones so as to not make all the cards dissapear and reappear
- Make window that holds robot cards scrollable and paginated sort of as to not load too many robot cards at once
- Make the base layout for making a robot page and assigning quick action to robot cards so can easily create frontend for custom robot
  - For the quick action, it could be a fetch to /api/\[robotID\]/quick_action and this fetch makes the server call a `quick_action` method of the robot handler